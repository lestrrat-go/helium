<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
  <xsl:template name="xsl:initial-template" >
    <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
      <out>
        <xsl:value-of
          select="format-number(avg((. treat as document-node(element(ACCOUNT)))/*/transaction/@value), '#.000')"
        />
      </out>
    </xsl:source-document>
  </xsl:template>
    
</xsl:stylesheet>