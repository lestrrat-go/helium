<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:ls="http://loan.shark.com/"
    exclude-result-prefixes="ls xs"
    xpath-default-namespace="http://loan.shark.com/"
    version="3.0">
       
  
    <!-- within xsl:stream, use xsl:iterate and xsl:break to process the subtree 
         (xsl:break can terminate the parsing -->
     
         
    <xsl:output method="xml" indent="no" encoding="UTF-8" />
    
    <xsl:param name="sep" select="'|'" as="xs:string"/>
     
    <xsl:template name="main">
      <out>
        <xsl:source-document streamable="yes" href="../docs/loans.xml">
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
        </xsl:source-document>
      </out>
    </xsl:template>

         
</xsl:transform>

