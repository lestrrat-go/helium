<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#current-function?>
<!-- Test use of current() in the count pattern of xsl:number -->

<xsl:template match="*">
  <xsl:copy>
    <xsl:attribute name="nr">
      <!-- count nodes having the same name as their parent -->
      <xsl:number count="*[name()=name(current())]/*" level="any"/>
    </xsl:attribute>
    <xsl:apply-templates/>
  </xsl:copy>
</xsl:template>
 
</xsl:stylesheet>
