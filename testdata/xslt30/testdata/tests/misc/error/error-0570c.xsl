<?xml version="1.0" encoding="UTF-8"?>
<!--It is a type error if the supplied value of a variable cannot be
                     converted to the required type.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                version="2.0"
                exclude-result-prefixes="xs">

<!-- PURPOSE:  type-check contents of a template rule against the type of the context item
     (could be detected statically by inference from the match pattern, but dynamic reporting is also allowed) -->
     
<?spec xslt#errors?><?error XTTE0570?>

   <xsl:template match="/">
      <xsl:apply-templates select="*/text()"/>
      <xsl:message>***** error not detected! *****</xsl:message>
   </xsl:template>

   <xsl:template match="text()">
      <a>
         <b>
            <xsl:variable name="n" select="." as="element()"/>
            <xsl:value-of select="$n"/>
         </b>
      </a>
   </xsl:template>

</xsl:stylesheet>
