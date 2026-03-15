<?xml version="1.0"?> 
<?spec fo#func-position?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Exercise potential optimization when position() appears in a predicate. -->

  <xsl:output method="xml" indent="no" encoding="UTF-8"/>

  <xsl:template match="doc">
    <out>
      <xsl:apply-templates select="*"/>
    </out>
  </xsl:template>

  <xsl:template match="doc/a[position() mod 2 = 0]">
    <a-even/>
  </xsl:template>

  <xsl:template match="doc/a[position() mod 2 = 1]">
    <a-odd/>
  </xsl:template>
  
  <xsl:template match="doc/b[position() mod 2 = 0]">
    <b-even/>
  </xsl:template>
  
  <xsl:template match="doc/b[position() mod 2 = 1]">
    <b-odd/>
  </xsl:template>
  
  <xsl:template match="doc/c[position() mod 2 = 0]">
    <c-even/>
  </xsl:template>
  
  <xsl:template match="doc/c[position() mod 2 = 1]">
    <c-odd/>
  </xsl:template>

</xsl:stylesheet>
