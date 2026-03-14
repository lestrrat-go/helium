<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Tests xs:string and xs:untypedAtomic types in a sequence  -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:variable name="strvalues" as="item()*">
      <xslt:sequence select="(xs:untypedAtomic('untypedfoo'),xs:string('strfoo'),'literalfoo',doc/string)"/>
   </xslt:variable>

   <xslt:variable name="untypedvalues" as="xs:untypedAtomic*">
      <xslt:sequence select="(xs:untypedAtomic('untypedfoo'),doc/string)"/>
   </xslt:variable>

   <xslt:template match="/">
      <out>
         <xslt:text>
</xslt:text>
         <strout>
            <xslt:value-of select="$strvalues[0]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$strvalues[0] instance of xs:string"/>
         </strout>
         <xslt:text>
</xslt:text>
         <strout>
            <xslt:value-of select="$strvalues[1]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$strvalues[1] instance of xs:string"/>
         </strout>
         <xslt:text>
</xslt:text>
         <strout>
            <xslt:value-of select="$strvalues[2]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$strvalues[2] instance of xs:string"/>
         </strout>
         <xslt:text>
</xslt:text>
         <strout>
            <xslt:value-of select="$strvalues[3]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$strvalues[3] instance of xs:string"/>
         </strout>
         <xslt:text>
</xslt:text>
         <strout>
            <xslt:value-of select="$strvalues[4]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$strvalues[4] instance of xs:string"/>
         </strout>
         <xslt:text>
</xslt:text>
         <strout>
            <xslt:value-of select="$strvalues[5]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$strvalues[5] instance of xs:string"/>
         </strout>
         <xslt:text>
</xslt:text>
         <untypedout>
            <xslt:value-of select="$untypedvalues[0]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$untypedvalues[0] instance of xs:untypedAtomic"/>
         </untypedout>
         <xslt:text>
</xslt:text>
         <untypedout>
            <xslt:value-of select="$untypedvalues[1]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$untypedvalues[1] instance of xs:untypedAtomic"/>
         </untypedout>
         <xslt:text>
</xslt:text>
         <untypedout>
            <xslt:value-of select="$untypedvalues[2]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$untypedvalues[2] instance of xs:untypedAtomic"/>
         </untypedout>
         <xslt:text>
</xslt:text>
         <untypedout>
            <xslt:value-of select="$untypedvalues[3]"/>
            <xslt:text> | </xslt:text>
            <xslt:value-of select="$untypedvalues[3] instance of xs:untypedAtomic"/>
         </untypedout>
         <xslt:text>
</xslt:text>
      </out>  
   </xslt:template>
</xslt:transform>
