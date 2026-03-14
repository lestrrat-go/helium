<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">

                    <!--It is a type error if an
                        xsl:apply-templates instruction in a particular mode
                        selects an element or attribute whose type is xs:untyped or
                        xs:untypedAtomic when the typed attribute of that
                        mode specifies the value yes, strict, or lax.-->
                        
   <xsl:mode typed="lax"/>
   
   <xsl:variable name="e" as="element()">
     <foo xsl:validation="strip"/>
   </xsl:variable>
                        
   <xsl:template match="/">
      <out>
         <xsl:apply-templates select="$e"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
