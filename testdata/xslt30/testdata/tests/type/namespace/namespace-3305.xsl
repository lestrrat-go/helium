<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#creating-attributes?>
    <!-- Purpose: Put namespace attribute on xsl:attribute,
    but set it to null (which it would have been anyway). -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="/">
  <out>
    <xsl:attribute name="Attr0" namespace="">whatever</xsl:attribute>
  </out>
</xsl:template>

</xsl:stylesheet>