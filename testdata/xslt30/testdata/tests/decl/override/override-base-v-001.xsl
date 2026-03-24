<xsl:package 
  name="http://www.w3.org/xslt30tests/override-base-201"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
    <xsl:mode/>
  
    <xsl:variable name="v-public" as="xs:integer" visibility="public" select="$v-private - 1"/>
    
    <xsl:variable name="v-private" as="xs:integer" visibility="private" select="$v-base"/>
    
    <xsl:variable name="v-final" as="xs:integer" visibility="final" select="3"/>
    
    <xsl:variable name="v-abstract" as="xs:integer" visibility="abstract" />
    
    <xsl:variable name="v-base" as="xs:integer" visibility="public" select="2"/>
    
    <xsl:template name="go" visibility="public">
      <xsl:param name="node" as="node()"/>
      <go><xsl:apply-templates select="$node"/></go>
    </xsl:template>
    
    <xsl:template match="*[$v-public = 1]">
      <one/>
    </xsl:template> 
    
    <xsl:template match="*[$v-public != 1]">
      <not-one/>
    </xsl:template>   
    
  
  
</xsl:package>   