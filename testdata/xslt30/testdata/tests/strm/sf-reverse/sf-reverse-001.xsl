<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
    
    <!-- Streaming reverse(): grounded operand -->
    
    <xsl:template name="r-001" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/recursive.xml">
        <out>
          <xsl:value-of select="reverse(snapshot(/chapter)//section)/@id"/>
        </out>
      </xsl:source-document>
    </xsl:template>

    
</xsl:stylesheet>