<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Tests xs:boolean and xs:untypedAtomic types in a sequence -->

   <xslt:variable name="values" as="item()*">
      <xslt:sequence select="(xs:untypedAtomic('true'),doc/boolean,xs:boolean('true'),false())"/>
   </xslt:variable>

   <xslt:template match="/">
      <output xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <xslt:text>
</xslt:text>
         <out>
            <xslt:value-of select="$values[0]"/>
            <xslt:value-of select="$values[0] instance of xs:boolean"/>
         </out>
         <xslt:text>
</xslt:text>
         <out>
            <xslt:value-of select="$values[1]"/>
            <xslt:value-of select="$values[1] instance of xs:boolean"/>
         </out>
         <xslt:text>
</xslt:text>
         <out>
            <xslt:value-of select="$values[2]"/>
            <xslt:value-of select="$values[2] instance of xs:boolean"/>
         </out>
         <xslt:text>
</xslt:text>
         <out>
            <xslt:value-of select="$values[3]"/>
            <xslt:value-of select="$values[3] instance of xs:boolean"/>
         </out>
         <xslt:text>
</xslt:text>
         <out>
            <xslt:value-of select="$values[4]"/>
            <xslt:value-of select="$values[4] instance of xs:boolean"/>
         </out>
         <xslt:text>
</xslt:text>
         <out>
            <xslt:value-of select="$values[5]"/>
            <xslt:value-of select="$values[5] instance of xs:boolean"/>
         </out>
      </output>
   </xslt:template>
</xslt:transform>
