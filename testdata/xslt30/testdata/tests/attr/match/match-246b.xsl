<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
  <!-- Purpose: Test comments in patterns. -->
  
<xsl:param name="target" select="'b'"/>
<xsl:strip-space elements="*"/>  

  <xsl:template match="(:1:)/(:2:)" priority="100"><out><xsl:apply-templates select="//letters"/></out></xsl:template>

  <xsl:template match="letters (:1:)" priority="20">Z<xsl:next-match/></xsl:template>
  <xsl:template match="letters (:1:)(:2:)" priority="19">Y<xsl:next-match/></xsl:template>
  <xsl:template match="letters (:1:)[true()](:2:)" priority="18">X<xsl:next-match/></xsl:template>
  <xsl:template match="letters (:1:)[true()(:3:)]" priority="17">W<xsl:next-match/></xsl:template>
  <xsl:template match="(:1:)letters" priority="16">V<xsl:next-match/></xsl:template>
  <xsl:template match=" (:1:) letters" priority="15">U<xsl:next-match/></xsl:template>
  <xsl:template match=" (:1(:2:):) letters" priority="14">T<xsl:next-match/></xsl:template>
  
  <xsl:template match="letters" priority="9"/>
 

</xsl:stylesheet>
