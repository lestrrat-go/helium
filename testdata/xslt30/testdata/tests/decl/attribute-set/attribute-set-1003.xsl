<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" version="2.0">
  <xsl:attribute-set name="attributeSet1">
    <xsl:attribute name="attr1"/>
  </xsl:attribute-set>
  <xsl:attribute-set name="attributeSet2" use-attribute-sets="attributeSet">
    <xsl:attribute name="attr1"/>
  </xsl:attribute-set>
</xsl:stylesheet>