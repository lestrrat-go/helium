<?xml version="1.0"?> 

<!-- try021  -->

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema">

<!-- Test xsl:try -->


<xsl:template name="main">
  <output>
    <xsl:try>
      <xsl:result-document href="../SaxonResults/try/try021.out">
        <test>
          <xsl:call-template name="test"/>
        </test>
      </xsl:result-document>
      <xsl:catch errors="*">
        <ERROR message="Division by 0 error caught!"/>
      </xsl:catch>
    </xsl:try>
  </output>
</xsl:template>
  
  
  <xsl:template name="test">
    <xsl:param name="zero" select="0" as="xs:integer"/>
    <xsl:for-each select="1 to 10">
          <value>
          <xsl:value-of select="1 div $zero"/>
          </value>       
    </xsl:for-each>
  </xsl:template>


</xsl:stylesheet>
