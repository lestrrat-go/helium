<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<!-- tests the use of current() within xsl:key. See Saxon bug 520552. -->
<?spec xslt#key?>
  <xsl:template match="text()" />

  <xsl:key name="targets"
    match="target"
    use="string(namespace::*[name()=substring-before(current()/@ref,':')])" />

  <xsl:template match="/">
  <out><xsl:apply-templates/></out>
  </xsl:template>

  <xsl:template match="namespace">
    <xsl:text />Namespace "<xsl:value-of select="@value" />
    <xsl:text>" used in </xsl:text>
    <xsl:for-each select="key('targets', @value)">
      <xsl:value-of select="@name" />
      <xsl:if test="position() != last()">, </xsl:if>
    </xsl:for-each>
    <xsl:text>&#xA;</xsl:text>
  </xsl:template>


</xsl:stylesheet>
