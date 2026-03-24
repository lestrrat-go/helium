<xsl:stylesheet
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:xs="http://www.w3.org/2001/XMLSchema"
   version="2.0">
   
   <!-- Test use of non-BMP character as grouping separator -->

   <xsl:output method="xml" encoding="iso-8859-1"/>

   <xsl:template match="/">
     <out><xsl:number value="1234567" grouping-size="3" grouping-separator="&#x10100;"/></out>
   </xsl:template>

</xsl:stylesheet>