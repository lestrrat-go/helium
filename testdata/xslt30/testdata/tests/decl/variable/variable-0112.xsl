<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
                
<!-- Test required="yes" on a global parameter -->
  <?spec xslt#parameters?>
<?error ?>
<?same-as-1.0 no?> 

  <xsl:param name="p" required="yes"/>           
  
  <xsl:template match="/">
    <xsl:apply-templates select="*">
      <xsl:with-param name="mod" select="3"/>
    </xsl:apply-templates>
  </xsl:template>

  <xsl:template match="*">
    <xsl:param name="mod" select="1"/>
    <out result="{$p}"/>
  </xsl:template>
  
</xsl:stylesheet>
