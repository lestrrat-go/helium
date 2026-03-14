<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
                xmlns:xdt="http://www.w3.org/2005/04/xpath-datatypes">
                
                <?spec fo#func-prefix-from-QName?>
                <?spec fo#func-resolve-QName?>
                
<xsl:template match="xs:schema">
  <xsl:apply-templates select="xs:element" />
</xsl:template>
                
<xsl:template match="xs:element">
  <xsl:element name="{@name}" namespace="{/xs:schema/@targetNamespace}">
    <xsl:variable name="prefix" as="xs:string"
      select="string(prefix-from-QName(resolve-QName(@type, .)))" />
    <xsl:sequence select="namespace::*[name(.) = $prefix]" />
    <xsl:attribute name="xsi:type">
      <xsl:value-of select="@type" />
    </xsl:attribute>
    ...
  </xsl:element>
</xsl:template>

</xsl:stylesheet>