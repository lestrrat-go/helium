<?xml version='1.0'?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
>

<xsl:import-schema namespace="http://www.w3.org/1999/XSL/Transform"
                   schema-location="schema-for-xslt20.xsd"/>
                   
                   <?spec xpath#id-schema-element-test?>

<xsl:template match="/">
<out>
  <z><xsl:value-of select="count(//schema-element(xsl:instruction))"/></z>
  <xsl:for-each select="//schema-element(xsl:instruction)">
    <n><xsl:value-of select="name()"/></n>
  </xsl:for-each>
</out>
</xsl:template>  

</xsl:stylesheet>