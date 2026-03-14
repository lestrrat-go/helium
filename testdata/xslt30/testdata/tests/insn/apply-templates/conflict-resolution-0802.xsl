<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
 xpath-default-namespace="http://some.uri/" >

<?spec xslt#modes?>
<!-- Test xsl:templates mode="a b #default" (2.0 facility) -->

<xsl:template match="doc" >
   <out>
      <xsl:apply-templates select="foo" mode="a"/>
      <xsl:apply-templates select="foo" mode="b"/>
      <xsl:apply-templates select="foo" mode="#default"/>
   </out>
</xsl:template>
  
<xsl:template match="foo" mode="a b #default">
    <xsl:call-template name="common"/>
</xsl:template>

<xsl:template match="bar" mode="a">
    <a/>
</xsl:template>

<xsl:template match="bar" mode="b">
    <b/>
</xsl:template>

<xsl:template match="bar">
    <default/>
</xsl:template>

<xsl:template name="common">
    <xsl:apply-templates select="//bar" mode="#current"/>
</xsl:template>

</xsl:stylesheet>
