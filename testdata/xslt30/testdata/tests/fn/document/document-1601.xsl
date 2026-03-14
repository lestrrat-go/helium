<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
<!-- Test document() function: Provides multiple input sources. -->

<xsl:template match="catalog.entry">
   <out>
      <s1><xsl:apply-templates select="catalog.entry.id"/></s1>
      <s2><xsl:apply-templates 
           select="document(catalog.operator.pointer/url.reference/@url.string)/market.participant/business.identity.group/business.name"/></s2>
      <s3><xsl:apply-templates select="document('../document/doc08.xml')/market.participant/address.set/*"/></s3>
   </out>
</xsl:template>

<xsl:template match="location.in.street">
    <loc><xsl:apply-templates/></loc>
</xsl:template>

<xsl:template match="street">
    <st><xsl:apply-templates/></st>
</xsl:template>

<xsl:template match="city">
    <city><xsl:apply-templates/><xsl:text>, </xsl:text></city>
</xsl:template>

<xsl:template match="country.subentity">
    <co><xsl:apply-templates/><xsl:text> </xsl:text></co>
</xsl:template>

<xsl:template match="postcode">
    <pc><xsl:apply-templates/></pc>
</xsl:template>

<xsl:template match="telephone.number">
    <ph><xsl:apply-templates/></ph>
</xsl:template>
       
</xsl:stylesheet>