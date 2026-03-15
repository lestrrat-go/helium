<xsl:package
    name="http://www.w3.org/xslt30tests/package-923"  
    package-version="1.0.0"
    version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:t="urn:template"
    >
  
    <!-- tests error XTSE3025 to be raised with xsl:expose and abstract, if there's no declared visibility -->

    <xsl:template name="t:abstract" />
    
    <xsl:template name="xsl:initial-template" visibility="public">
      <not-ok/>
    </xsl:template>
    
    <xsl:expose visibility="abstract" component="*" names="t:*"/>

</xsl:package>   