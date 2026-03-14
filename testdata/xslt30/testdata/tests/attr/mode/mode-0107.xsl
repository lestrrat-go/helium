<?xml version="1.0"?> 
<?spec xslt#modes?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE:  Test apply-templates to the root of a temporary tree  -->
  <xsl:template match="/">
    <out>
      <xsl:apply-templates select="$x" mode="a"/>
    </out>
  </xsl:template>

  <xsl:template match="/" mode="a">
    <tree>
      <xsl:apply-templates/>
    </tree>
  </xsl:template>

  <xsl:template match="x">
    <x/>
  </xsl:template>

  <xsl:variable name="x">
    <x/>
  </xsl:variable>

</xsl:stylesheet>
