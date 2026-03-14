<?xml version='1.0'?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  >
  
<?spec xpath#id-schema-element-test?>
  
<xsl:import-schema namespace="http://www.w3.org/1999/XSL/Transform"
                   schema-location="schema-for-xslt20.xsd"/>

<xsl:template match="/">
  <z><xsl:value-of select="count(//schema-element(xsl:instruction))"/></z>

</xsl:template>  

</xsl:stylesheet>