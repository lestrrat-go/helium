<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">

                <!-- It is a dynamic error if the value of the first
                     argument to the accumulator-before or accumulator-after function is not a valid
                     EQName, or if there is no namespace declaration in scope
                     for the prefix of the QName, or if the name obtained by expanding the QName is not
                     the same as the expanded name of any xsl:accumulator declaration
                     visible in the package in which the function
                     call appears. If the processor is able to detect the error statically (for
                     example, when the argument is supplied as a string literal), then the processor
                     may optionally signal this as a static error.-->
                     
   <xsl:param name="n" required="yes" as="xs:string"/>
   <xsl:variable name="v"><x/></xsl:variable>
   
   <xsl:template name="main">
      <out>
         <xsl:value-of select="$v/x/accumulator-after($n)"/>
      </out>
   </xsl:template>
   
</xsl:stylesheet>
