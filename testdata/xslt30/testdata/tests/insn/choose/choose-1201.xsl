<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#xsl-choose?>
    <!-- Purpose: Test that xsl:choose can be nested. -->

<xsl:template match="/doc">
  <out>
    <xsl:for-each select=".//title">
      <xsl:choose>
        <xsl:when test=".='Level A'">*A+</xsl:when>
        <xsl:when test=".='Level B'">B+</xsl:when>
        <xsl:when test=".='Level C'">
          <xsl:choose><!-- When on a C, look ahead -->
            <xsl:when test="name(following-sibling::*[1])='d'">C+</xsl:when>
            <xsl:when test="name(../following-sibling::*[1])='c'">C:</xsl:when>
            <xsl:when test="name(../../following-sibling::*[1])='b'">C-</xsl:when>
            <xsl:otherwise>!Bad tree!</xsl:otherwise>
          </xsl:choose>
        </xsl:when>
        <xsl:when test=".='Level D'">
          <xsl:choose><!-- When on a D, look ahead -->
            <xsl:when test="name(following-sibling::*[1])='e'">D+</xsl:when>
            <xsl:when test="name(../following-sibling::*[1])='d'">D:</xsl:when>
            <xsl:otherwise><!-- We're backing up, but how far? -->
              <xsl:choose>
                <xsl:when test="name(../../following-sibling::*[1])='c'">D-</xsl:when>
                <xsl:otherwise>D|</xsl:otherwise>
              </xsl:choose>
            </xsl:otherwise>
          </xsl:choose>
        </xsl:when>
        <xsl:when test=".='Level E'">E-</xsl:when>
        <xsl:otherwise>TREE: </xsl:otherwise>
      </xsl:choose>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>
