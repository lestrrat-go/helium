<?xml version="1.0" encoding="UTF-8"?>
<!--A type error occurs if a
                              type or validation attribute is defined
                           (explicitly or implicitly) for an instruction that constructs a new
                           attribute node, if the effect of this is to cause the attribute value to
                           be validated against a type that is derived from, or constructed by list
                           or union from, the primitive types xs:QName or
                              xs:NOTATION.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTTE1545?>


  <xsl:template name="main">
    
      <out> 
         <xsl:attribute name="n" type="xs:QName">my:name</xsl:attribute>      
      </out>
  </xsl:template>
  




</xsl:stylesheet>
