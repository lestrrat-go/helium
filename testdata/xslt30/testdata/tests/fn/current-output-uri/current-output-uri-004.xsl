<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <!-- current-output-uri() in a tunnel param -->

  <xs:table>
    <xs:row>
       <xs:column value="12"/>
    </xs:row>
  </xs:table>
  
  <xsl:strip-space elements="*"/>  
  
  <xsl:template match="/">
    <xsl:apply-templates>
      <xsl:with-param name="base-output-uri" tunnel="yes" select="current-output-uri()"/>
    </xsl:apply-templates>
  </xsl:template>
  
  <xsl:template match="xs:row">
    <xsl:result-document href="row{position()}.xml">
      <row><xsl:apply-templates/></row>
    </xsl:result-document>
  </xsl:template>
  
  <xsl:template match="xs:column">
    <xsl:param name="base-output-uri" tunnel="yes"/>
    <col base="{$base-output-uri}"/>
  </xsl:template>

</xsl:stylesheet>
