<xsl:package 
  name="http://www.w3.org/xslt30tests/override-201"
  package-version="1.0.1"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  <!-- Test that global variables referenced in a match pattern can be overridden -->
  
  <xsl:use-package
    name="http://www.w3.org/xslt30tests/override-base-201"
    package-version="1.0.0">
  
    <xsl:override>
  
      <xsl:variable name="v-public" as="xs:integer" visibility="public" select="13"/>
    
      <xsl:variable name="v-abstract" as="xs:integer" visibility="public" select="93"/>
      
    </xsl:override>  
    
  </xsl:use-package>
  
  
   
    <xsl:template name="main" visibility="public">
      <xsl:variable name="doc"><a><b/></a></xsl:variable>
      <xsl:call-template name="go">
        <xsl:with-param name="node" select="$doc//b"/>
      </xsl:call-template>
    </xsl:template>
    
 
  
</xsl:package>   