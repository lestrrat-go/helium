<?xml version="1.0"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Show discrepancies between number and position. -->
  
  <!-- Elaboration: While xsl:number always shows the number of the chapter among all chapters,
       position() in the outer case is the position of the chapter among all children of doc,
       position() in the inner case (repeat mode) is the position of the chapter/text within the select set
       that contains the chapter and all its siblings. -->

<xsl:template match="doc">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="chapter">
  <xsl:element name="chap">
    <xsl:attribute name="number">
      <xsl:number/>
    </xsl:attribute>
    <xsl:attribute name="position">
      <xsl:value-of select="position()"/>
    </xsl:attribute>
    <note><xsl:value-of select="note[1]"/></note>
    <xsl:apply-templates select="../node()" mode="repeat"/>
  </xsl:element>
</xsl:template>

<xsl:template match="chapter" mode="repeat">
  <xsl:element name="sel">
    <xsl:attribute name="number">
      <xsl:number/>
    </xsl:attribute>
    <xsl:attribute name="position">
      <xsl:value-of select="position()"/>
    </xsl:attribute>
  </xsl:element>
</xsl:template>

<xsl:template match="text()" mode="repeat">
  <xsl:element name="text">
    <xsl:attribute name="number">
      <xsl:number/>
    </xsl:attribute>
    <xsl:attribute name="position">
      <xsl:value-of select="position()"/>
    </xsl:attribute>
  </xsl:element>
</xsl:template>

</xsl:stylesheet>
