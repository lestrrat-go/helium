<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:ls="http://loan.shark.com/"
    exclude-result-prefixes="ls xs"
    xpath-default-namespace="http://loan.shark.com/"
    version="3.0">
       
  
    <!-- within a streaming template, use xsl:iterate and xsl:break to process the subtree -->
     
    <xsl:mode name="s" streamable="yes"/>
         
    <xsl:output method="xml" indent="no" encoding="UTF-8" />
    
    <xsl:param name="sep" select="'|'" as="xs:string"/>
     
    <xsl:template name="main" match="/">
      <out>
        <xsl:source-document streamable="yes" href="../docs/loans.xml">
        	<xsl:apply-templates select="*/ACERequest" mode="s"/>
        </xsl:source-document>
      </out>
    </xsl:template>
    
    <xsl:template match="ACERequest" mode="s">
      <loans>
        <xsl:iterate select="outermost(.//UnpaidPrincipalBalanceAmount)">
          <xsl:param name="acc" as="xs:decimal" select="0.00"/>
          <xsl:variable name="loan" select="xs:decimal(.)"/>
          <xsl:variable name="newAcc" as="xs:decimal" select="$acc + $loan"/>
          <xsl:choose>
             <xsl:when test="$newAcc > 3000000">
                  <xsl:break>
                      <broken/>
                  </xsl:break>
             </xsl:when>
             <xsl:otherwise>
                  <loan value="{$loan}" cumulativeValue="{$newAcc}"/>
                  <xsl:next-iteration>
                     <xsl:with-param name="acc" select="$newAcc"/>
                  </xsl:next-iteration>
             </xsl:otherwise>
          </xsl:choose>
        </xsl:iterate>
      </loans>
    </xsl:template>
         
</xsl:transform>

