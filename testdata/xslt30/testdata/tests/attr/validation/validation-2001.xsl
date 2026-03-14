<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:t="http://ns.example.com/val12/"
>

<!-- show patterns matching against nilled and unnilled elements in the source -->
<?spec xpath#id-element-test?>
<?spec xslt#patterns?>

<xsl:import-schema namespace="http://ns.example.com/val12/" schema-location="validation-12.xsd"/>
<xsl:template name="main">
<z>
  <xsl:apply-templates select="doc('validation-20.xml')/*"/>
  <xsl:apply-templates select="doc('validation-20a.xml')/*"/>
</z>
</xsl:template>

<xsl:template match="element(t:test, t:testType?)" priority="2">
  <a>nilled</a>
</xsl:template>

<xsl:template match="element(t:test, t:testType)" priority="3">
  <a>not nilled</a>
</xsl:template>

</xsl:stylesheet>
