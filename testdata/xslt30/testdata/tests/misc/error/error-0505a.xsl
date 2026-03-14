<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

                 <!--It is a type error if the result of
                     evaluating the sequence
                     constructor cannot be converted to the required type.-->
   <xsl:template name="main">
      <out>
         <xsl:call-template name="t"/>
      </out>
   </xsl:template>
   
   <xsl:template name="t" as="xs:integer">
     <xsl:sequence select="current-date()"/>
   </xsl:template>
   
</xsl:stylesheet>
