<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#built-in-rule?>
<!-- tests whether parameters to a built-in template are passed through in 1.0 mode -->
<!-- (note this isn't a legal 1.0 stylesheet because it uses an RTF as a node-set) -->
<?same-as-1.0 no?>

<!-- the spec has changed several times. Current state is that they ARE passed through - gives answer 42 -->

<xsl:template match="top">
  <!-- ** changes in 8.2 - built-in templates don't pass parameters in 1.0 mode ** -->
  <xsl:variable name="x">
    <inner/>
  </xsl:variable>
  <xsl:apply-templates select="$x">
    <xsl:with-param name="hi" select="42"/>
  </xsl:apply-templates>
</xsl:template>

<xsl:template match="inner">
  <xsl:param name="hi" select="21"/>
  <z><xsl:value-of select="$hi"/></z>
</xsl:template>

</xsl:stylesheet>