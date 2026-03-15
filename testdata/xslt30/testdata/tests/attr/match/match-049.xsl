<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs"
  version="2.0">
  
  <!-- From a bug report submitted by Christian Roth on the saxon-help mailing list, 2010-04-02 -->

  <xsl:template match="*[name() eq name(current())] | text()">
    <node name="{name()}">
      <xsl:apply-templates/>
    </node>
  </xsl:template>

</xsl:stylesheet>