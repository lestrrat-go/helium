<xsl:package 
  name="http://www.w3.org/xslt30tests/override-t-014"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/override-base-101"
  exclude-result-prefixes="xs p">
  
  <xsl:use-package
    name="http://www.w3.org/xslt30tests/override-t-014-lib"
    package-version="1.0.0">
    
    <xsl:accept component="template" names="*" visibility="hidden"/>
  
    <xsl:override>
  
      <xsl:template name="t" visibility="public">
        <ttt override="yes"/>
      </xsl:template>

    </xsl:override>  
    
  </xsl:use-package>
  
  <xsl:use-package
    name="http://www.w3.org/xslt30tests/override-t-014-lib"
    package-version="1.0.0">
    
    <xsl:accept component="template" names="*" visibility="hidden"/>
    
    <xsl:override>
      
      <xsl:template name="u" visibility="public">
        <uuu override="yes"/>
      </xsl:template>
      
    </xsl:override>  
    
  </xsl:use-package>
  
 
   
    <xsl:template name="main" visibility="public">
    <out>
      <xsl:call-template name="t"/>
      <xsl:call-template name="u"/>
    </out>  
    </xsl:template>
    

  
</xsl:package>   