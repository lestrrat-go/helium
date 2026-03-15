<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-choose?>
    <!-- Purpose: An optimizable xsl:choose. -->

<xsl:template match="/doc">
  <out>
    <xsl:for-each select=".//*">
      <xsl:choose>
        <xsl:when test="string-length(.) = 0">A</xsl:when>
        <xsl:when test="string-length(.) = 1">B</xsl:when>
        <xsl:when test="string-length(.) = 2">C</xsl:when>
        <xsl:when test="string-length(.) = 3">D</xsl:when>
        <xsl:when test="string-length(.) = 4">E</xsl:when>
        <xsl:when test="string-length(.) = 5">F</xsl:when>
        <xsl:when test="string-length(.) = 6">G</xsl:when>
        <xsl:when test="string-length(.) = 7">H</xsl:when>
        <xsl:when test="string-length(.) = 8">I</xsl:when>
        <xsl:when test="string-length(.) = 9">J</xsl:when>
        <xsl:when test="string-length(.) = 10">K</xsl:when>
        <xsl:when test="string-length(.) = 11">L</xsl:when>
        <xsl:when test="string-length(.) = 12">M</xsl:when>
        <xsl:when test="string-length(.) = 13">N</xsl:when>
        <xsl:when test="string-length(.) = 14">O</xsl:when>
        <xsl:when test="string-length(.) = 15">P</xsl:when>
        <xsl:when test="string-length(.) = 16">Q</xsl:when>
        <xsl:when test="string-length(.) = 17">R</xsl:when>
        <xsl:when test="string-length(.) = 18">S</xsl:when>
        <xsl:when test="string-length(.) = 19">T</xsl:when>
        <xsl:when test="string-length(.) = 20">U</xsl:when>
        <xsl:when test="string-length(.) = 21">V</xsl:when>
        <xsl:when test="string-length(.) = 22">W</xsl:when>
        <xsl:when test="string-length(.) = 23">X</xsl:when>
        <xsl:when test="string-length(.) = 24">Y</xsl:when>
        <xsl:when test="string-length(.) = 25">Z</xsl:when>
        <xsl:otherwise>*</xsl:otherwise>
      </xsl:choose>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
