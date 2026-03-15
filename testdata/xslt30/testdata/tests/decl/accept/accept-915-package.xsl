<!-- xsl:expose - error, cannot use #unnamed mode -->

<xsl:package
  name="http://www.w3.org/xslt30tests/package-915"  
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:m="urn:mode">
  
  <xsl:mode/>
  <xsl:mode name="m1"/>
  <xsl:mode name="m:m2"/>
  

    <xsl:template name="xsl:initial-template">
      <not-ok/>
    </xsl:template>
    
    <!-- tests error XTSE0020 to be raised on invalid token -->
    <xsl:expose visibility="public" component="mode" names="*:m2 #unnamed"/>

</xsl:package>   