<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-choose?>
    <!-- Purpose: An optimizable xsl:choose using a case-blind collation. -->

<xsl:template match="/doc">
  <out>
    <xsl:for-each select=".//*">
      <xsl:choose default-collation="http://www.w3.org/2005/xpath-functions/collation/html-ascii-case-insensitive http://www.w3.org/2013/collation/UCA?lang=en;strength=secondary ">
        <xsl:when test="name(.) = 'A'">A</xsl:when>
        <xsl:when test="name(.) = 'B'">B</xsl:when>
        <xsl:when test="name(.) = 'C'">C</xsl:when>
        <xsl:when test="name(.) = 'D'">D</xsl:when>
        <xsl:when test="name(.) = 'E'">E</xsl:when>
        <xsl:when test="name(.) = 'F'">F</xsl:when>
        <xsl:when test="name(.) = 'G'">G</xsl:when>
        <xsl:when test="name(.) = 'H'">H</xsl:when>
        <xsl:when test="name(.) = 'I'">I</xsl:when>
        <xsl:when test="name(.) = 'J'">J</xsl:when>
        <xsl:when test="name(.) = 'K'">K</xsl:when>
        <xsl:when test="name(.) = 'L'">L</xsl:when>
        <xsl:when test="name(.) = 'M'">M</xsl:when>
        <xsl:when test="name(.) = 'N'">N</xsl:when>
        <xsl:when test="name(.) = 'O'">O</xsl:when>
        <xsl:when test="name(.) = 'P'">P</xsl:when>
        <xsl:when test="name(.) = 'Q'">Q</xsl:when>
        <xsl:when test="name(.) = 'R'">R</xsl:when>
        <xsl:when test="name(.) = 'S'">S</xsl:when>
        <xsl:when test="name(.) = 'T'">T</xsl:when>
        <xsl:when test="name(.) = 'U'">U</xsl:when>
        <xsl:when test="name(.) = 'V'">V</xsl:when>
        <xsl:when test="name(.) = 'W'">W</xsl:when>
        <xsl:when test="name(.) = 'X'">X</xsl:when>
        <xsl:when test="name(.) = 'Y'">Y</xsl:when>
        <xsl:when test="name(.) = 'Z'">Z</xsl:when>
        <xsl:otherwise>*</xsl:otherwise>
      </xsl:choose>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
