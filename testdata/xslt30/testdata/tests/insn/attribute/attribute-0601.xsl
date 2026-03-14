<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
      xmlns:ped="http://ped.test.com">

<?spec xslt#creating-attributes?>
    <!-- Purpose: The namespace attribute is interpreted as an attribute value template. -->

<xsl:template match="/">
 <root>
  <Out>
	<xsl:attribute name="a:{docs/a}" namespace="{docs/a/@href}">Hello</xsl:attribute>
	<xsl:attribute name="b:{docs/b}" namespace="{docs/b/@href}">Whatsup</xsl:attribute>
	<xsl:attribute name="c:Attr2" namespace="http://ped.test.com">Goodbye</xsl:attribute>
  </Out>
 </root>
</xsl:template>

</xsl:stylesheet>
