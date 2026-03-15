<xsl:transform version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
  
    <xsl:param name="test" as="xs:boolean" select="true()"/>
  
  <!-- Streaming xsl:assert tests with test evaluating to false -->
   
  <!-- within xsl:source-document, use xsl:assert: atomic values, consuming -->
  
  <xsl:template name="xsl:initial-template">
    <out>
      <xsl:source-document streamable="yes" href="../docs/transactions.xml">
        <xsl:for-each select="account/transaction[@value &lt; 0]/@value">
          <xsl:assert test="number(.) ge 0" error-code="XX99"><xsl:sequence select="'Found negative value: ' || data(.)"/></xsl:assert>
        </xsl:for-each>
      </xsl:source-document>
    </out>
  </xsl:template>
  

  
</xsl:transform>  