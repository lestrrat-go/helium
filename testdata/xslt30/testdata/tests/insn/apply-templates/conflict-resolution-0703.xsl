<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
 xpath-default-namespace="http://some.uri/" >

<?spec xslt#unprefixed-qnames?>
<!-- Testing xpath-default-namespace doesn't apply to attributes -->
<xsl:template match="doc" >
   <out>
      <xsl:apply-templates select="foo"/>
   </out>
</xsl:template>
  
<xsl:template match="foo">
    <xsl:text>foo</xsl:text>
    <xsl:apply-templates select="@test"/>
</xsl:template>

<xsl:template match="@test">
    <xsl:text/>"<xsl:value-of select="."/>"<xsl:text/>
</xsl:template>

</xsl:stylesheet>
