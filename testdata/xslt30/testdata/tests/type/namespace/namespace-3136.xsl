<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-element?>
    <!-- Purpose: Test resetting prefix and URI to what they were anyway. -->

<xsl:template match = "/">
  <out xmlns:baz="http://xyz/">
    <xsl:element namespace="http://xyz/" name="baz:foo">
      <baz:yyy/>
    </xsl:element>
  </out>
</xsl:template>

</xsl:stylesheet>


