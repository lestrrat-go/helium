<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <?spec xslt#document?>
    <!-- Purpose: Try sorting and numbering nodes from two other documents. -->
  <!-- If we didn't sort, we couldn't guarantee the order in which the documents would be read. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="count(document(a)//body)"/><xsl:text> body nodes:
</xsl:text>
    <xsl:apply-templates select="document(a)//body">
      <xsl:sort select="substring-after(.,'-')"/>
    </xsl:apply-templates>
  </out>
</xsl:template>

<xsl:template match="body">
  <xsl:value-of select="position()"/><xsl:text>. </xsl:text>
  <xsl:value-of select="."/><xsl:text>
</xsl:text>
</xsl:template>

</xsl:stylesheet>