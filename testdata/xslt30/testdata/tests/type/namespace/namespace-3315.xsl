<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#lre-namespaces?>
    <!-- Purpose: Test exclude-result-prefixes, attribute level -->

<xsl:template match="/">
  <out xmlns:anamespace="http://foo.com/"  xsl:exclude-result-prefixes="anamespace">
    <p>
      <xsl:attribute name="test" namespace="http://foo2.com/">true</xsl:attribute>
    </p>
  </out>
</xsl:template>

</xsl:stylesheet>
