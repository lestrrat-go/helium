<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Apply namespaces to attributes -->

  <xsl:template match="/">
    <out xmlns:anamespace="http://foo.com/">
      <p>
        <xsl:attribute name="Attr1" namespace="http://foo.com/">true</xsl:attribute>
      </p>
      <p>
        <xsl:attribute name="two:Attr2" namespace="http://baz.com/">true</xsl:attribute>
      </p>
      <!-- This 3rd case is a base line and should not have associated namespace -->
      <p>
        <xsl:attribute name="Attr3">true</xsl:attribute>
      </p>
    </out>
  </xsl:template>

</xsl:stylesheet>
