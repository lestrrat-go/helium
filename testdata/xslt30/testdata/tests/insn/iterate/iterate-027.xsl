<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <!-- group adjacent elements (without streaming) -->
  
  <xsl:template name="main">
    <account>
      <xsl:iterate select="doc('transactions.xml')/account/transaction">
      <xsl:param name="group" as="element(transaction)*" select="()"/>
      <xsl:param name="currentDate" as="xs:date?" select="()"/>
        <xsl:on-completion>
          <final-daily-transactions date="{$currentDate}">
            <xsl:copy-of select="$group"/>
          </final-daily-transactions>
        </xsl:on-completion> 
        <xsl:choose>
          <xsl:when test="xs:date(@date) eq $currentDate or empty($group)">
            <xsl:next-iteration>
              <xsl:with-param name="currentDate" select="@date"/>
              <xsl:with-param name="group" select="($group, .)"/>
            </xsl:next-iteration>
          </xsl:when>
          <xsl:otherwise>
            <daily-transactions date="{$currentDate}">
              <xsl:copy-of select="$group"/>
            </daily-transactions>
            <xsl:next-iteration>
              <xsl:with-param name="group" select="."/>
              <xsl:with-param name="currentDate" select="@date"/>
            </xsl:next-iteration>            
          </xsl:otherwise>
        </xsl:choose>       
      </xsl:iterate>
    </account>
  </xsl:template>
  
  
</xsl:stylesheet>