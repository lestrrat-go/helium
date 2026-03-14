<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
    <!-- Purpose: Test for resetting of a prefixed namespace by a LRE. -->

<xsl:template match = "/">
  <out xmlns:baz="http://xyz/">
    <xsl:element namespace="http://abc/" name="baz:foo">
      <baz:yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>


