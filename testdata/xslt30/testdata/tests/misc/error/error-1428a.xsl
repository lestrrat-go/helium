<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

            <!--  It is a non-recoverable dynamic
                  error if the argument passed to the type-available function does
                  not evaluate to a string that is a valid EQName, or if the value is a lexical QName
                  with a prefix for which no namespace declaration is present in the static context. If the processor is
                  able to detect the error statically (for example, when the argument is
                  supplied as a string literal), then the processor may
                  optionally signal this as a static error. -->
                  
   <xsl:param name="t" select="'foo::bar'"/>
   
   <xsl:template name="main">
      <out>
         <xsl:value-of select="type-available($t)"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
