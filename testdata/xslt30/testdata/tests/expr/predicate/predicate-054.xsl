<?xml version="1.0"?>
<?spec xpath#id-predicates?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- An opportunity for an optimizer to reorder predicates -->

  <xsl:template match="foo">
    <out>
      <xsl:apply-templates select="bar"/>
    </out>
  </xsl:template>

  <xsl:template match="foo[not(following::thwunk/preceding::thwank)]/bar[@a='1']">
    <bar seq="{@seq}"/>
  </xsl:template>
  
  <xsl:template match="foo/bar[following::thwunk/preceding::thwank]">
    <wrong/>
  </xsl:template>
  
  <xsl:template match="bar"/>


</xsl:stylesheet>
