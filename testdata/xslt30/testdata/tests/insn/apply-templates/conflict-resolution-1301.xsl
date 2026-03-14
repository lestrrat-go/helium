<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:fo="http://www.w3.org/1999/XSL/Format"
  xmlns="http://www.w3.org/1999/xhtml"
>

<?spec xslt#defining-template-rules?>

<xsl:output method="xml"
  omit-xml-declaration="no" encoding="ISO-8859-1"/>

<xsl:template match="/">
  <fo:root>
    <fo:layout-master-set>
      <fo:simple-page-master page-height="11in"
                             page-width="8.5in"
                             master-name="all-pages"
                             margin-left="1in"
                             margin-right="1in"
                             margin-top="50pt"
                             margin-bottom="50pt">
        <fo:region-body margin-bottom="25pt"
                        margin-top="25pt"/>
        <fo:region-after extent="25pt"/>
      </fo:simple-page-master>
    </fo:layout-master-set>
    <xsl:apply-templates/>
  </fo:root>
</xsl:template>

<xsl:template match="doc">
  <fo:page-sequence master-reference="all-pages" font-family="serif"
      font-size="12pt" line-height="13pt">
    <fo:flow flow-name="xsl-region-body">
      <xsl:apply-templates/>
    </fo:flow>
  </fo:page-sequence>
</xsl:template>

<xsl:template match="simplelist">
  <fo:block>
    <xsl:apply-templates/>
  </fo:block>
</xsl:template>

<xsl:template match="member[position()=last()]">
  <fo:block text-align="justify" color="blue" pos="{position()}" last="{last()}">
    <xsl:apply-templates/>
  </fo:block>
</xsl:template>

<xsl:template match="member[position()&lt;last()]">
  <fo:block text-align="justify" color="black" pos="{position()}" last="{last()}">
    <xsl:apply-templates/>
  </fo:block>
</xsl:template>

<xsl:template match="*" priority="-1">
  <fo:block color="red" pos="{position()}" last="{last()}">
    <xsl:apply-templates/>
  </fo:block>
</xsl:template>

</xsl:stylesheet>
