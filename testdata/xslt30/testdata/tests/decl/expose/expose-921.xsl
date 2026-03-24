<xsl:package
    name="http://www.w3.org/xslt30tests/package-921"  
    package-version="1.0.0"
    version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  
    <!-- tests error XTSE3025 to be raised for xsl:mode, trying to expose as abstract -->

    <xsl:mode name="m1"/>

    <xsl:template name="xsl:initial-template" visibility="public">
      <not-ok/>
    </xsl:template>
    
    <xsl:expose visibility="abstract" component="mode" names="m1"/>

</xsl:package>   