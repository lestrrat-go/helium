<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#creating-attributes?>
    <!-- Purpose: Try to put un-namespaced attribute on namespaced element. -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="/">
  <out xmlns:foo="http://foo.com/">
    <foo:pq>
      <xsl:attribute name="Attr1" namespace="">true</xsl:attribute>
    </foo:pq>
  </out>
</xsl:template>

</xsl:stylesheet>