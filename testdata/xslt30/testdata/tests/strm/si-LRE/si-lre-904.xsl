<xsl:transform version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>
   
  <!-- LRE referring to a streamable attribute set with streamable=" true " -->
  
  <xsl:attribute-set name="as-5" streamable=" true ">
    <xsl:attribute name="x" select="1"/>
    <xsl:attribute name="y" select="2"/>
  </xsl:attribute-set> 
  
  <xsl:template name="cy-904" use-when="$RUN">
    <out>
      <xsl:source-document streamable="yes" href="../docs/citygml.xml">
        <xsl:for-each select="*">
          <dddd xsl:use-attribute-sets="as-5">
            <xsl:sequence select="*" />
          </dddd>
        </xsl:for-each>  
      </xsl:source-document>
    </out>
  </xsl:template>
  
</xsl:transform>  