<?xml version="1.0" encoding="UTF-8"?>
<!--It is a type error if an
                              [xsl:]type attribute is defined for a constructed element
                           or attribute, and the outcome of schema validity assessment against that
                           type is that the validity property of that element or
                           attribute information item is other than valid.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTTE1540?>


  <xsl:template name="main">
    
      <out> 
         <e xsl:type="xs:date">2006-02-31</e>      
      </out>
  </xsl:template>
  




</xsl:stylesheet>
