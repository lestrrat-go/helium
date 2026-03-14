<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-choose?>
    <!-- Purpose: An optimizable xsl:choose. -->

<xsl:template match="/doc">
  <out>
    <xsl:for-each select=".//*">
      <xsl:choose>
        <xsl:when test="name(.) = 'a'">A</xsl:when>
        <xsl:when test="name(.) = 'b'">B</xsl:when>
        <xsl:when test="name(.) = 'c'">C</xsl:when>
        <xsl:when test="name(.) = 'd'">D</xsl:when>
        <xsl:when test="name(.) = 'e'">E</xsl:when>
        <xsl:when test="name(.) = 'f'">F</xsl:when>
        <xsl:when test="name(.) = 'g'">G</xsl:when>
        <xsl:when test="name(.) = 'h'">H</xsl:when>
        <xsl:when test="name(.) = 'i'">I</xsl:when>
        <xsl:when test="name(.) = 'j'">J</xsl:when>
        <xsl:when test="name(.) = 'k'">K</xsl:when>
        <xsl:when test="name(.) = 'l'">L</xsl:when>
        <xsl:when test="name(.) = 'm'">M</xsl:when>
        <xsl:when test="name(.) = 'n'">N</xsl:when>
        <xsl:when test="name(.) = 'o'">O</xsl:when>
        <xsl:when test="name(.) = 'p'">P</xsl:when>
        <xsl:when test="name(.) = 'q'">Q</xsl:when>
        <xsl:when test="name(.) = 'r'">R</xsl:when>
        <xsl:when test="name(.) = 's'">S</xsl:when>
        <xsl:when test="name(.) = 't'">T</xsl:when>
        <xsl:when test="name(.) = 'u'">U</xsl:when>
        <xsl:when test="name(.) = 'v'">V</xsl:when>
        <xsl:when test="name(.) = 'w'">W</xsl:when>
        <xsl:when test="name(.) = 'x'">X</xsl:when>
        <xsl:when test="name(.) = 'y'">Y</xsl:when>
        <xsl:when test="name(.) = 'z'">Z</xsl:when>
        <xsl:otherwise>*</xsl:otherwise>
      </xsl:choose>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
