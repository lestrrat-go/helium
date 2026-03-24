<?xml version="1.0" encoding="UTF-8"?>
<!--It is a type error if the supplied value of a variable cannot be
                     converted to the required type.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                version="2.0">

  <!-- PURPOSE: Test declaring a variable with incorrect type "element/attribute of type X". -->
<?spec xslt#errors?><?error XTSE0010?>

   <xsl:variable name="tree">
      <e1>
         <xsl:attribute name="id" type="xs:ID">A001</xsl:attribute>
      </e1>
      <e2 xsl:type="xs:NMTOKEN">A002</e2>
   </xsl:variable>

   <xsl:variable name="v1" select="$tree/e1/@id" as="attribute(*, xs:ID)"/>
   <xsl:variable name="v2" select="$tree/e2" as="element(*, xs:NMTOKEN)"/>            
   <xsl:variable name="v3" select="$tree/e2" as="element(*, xs:integer)"/>    
  <xsl:template match="/">
      <out val="{($v1, $v2, $v3)}"/>
  </xsl:template>

</xsl:stylesheet>
