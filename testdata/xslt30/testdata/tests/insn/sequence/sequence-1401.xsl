<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Tests xs:integer, xs:double. xs:decimal and xs:untypedAtomic types in a sequence.
  				Tests explicit items, literals, nodes, function 
  				conversion rules (casting from xs:untypedAtomic and numeric promotion) 
  				and Checks values and types of elements in sequence. -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:variable name="values" as="item()*">
      <xslt:sequence select="(xs:integer(1),xs:decimal(1.52), xs:double(1.52),xs:untypedAtomic('2.5E1'), xs:untypedAtomic('2'),3,4,doc/integer)"/>
   </xslt:variable>

   <xslt:template match="/">
	     <out>
         <xslt:text>
</xslt:text>
         <in>
            <xslt:value-of select="$values[0]"/>
            <xslt:text>|</xslt:text>
            <xslt:value-of select="$values[0] instance of xs:integer"/>
         </in>
         <xslt:text>
</xslt:text>
         <in>
            <xslt:value-of select="$values[1]"/>
            <xslt:text>|</xslt:text>
            <xslt:value-of select="$values[1] instance of xs:integer"/>
         </in>
         <xslt:text>
</xslt:text>
         <in>
            <xslt:value-of select="$values[2]"/>
            <xslt:text>|</xslt:text>
            <xslt:value-of select="$values[2] instance of xs:decimal"/>
         </in>
         <xslt:text>
</xslt:text>
         <in>
            <xslt:value-of select="$values[3]"/>
            <xslt:text>|</xslt:text>
            <xslt:value-of select="$values[3] instance of xs:double"/>
         </in>
         <xslt:text>
</xslt:text>
         <in>
            <xslt:value-of select="$values[4]"/>
            <xslt:text>|</xslt:text>
            <xslt:value-of select="$values[4] instance of xs:double"/>
         </in>
         <xslt:text>
</xslt:text>
         <in>
            <xslt:value-of select="$values[5]"/>
            <xslt:text>|</xslt:text>
            <xslt:value-of select="$values[5] instance of xs:integer"/>
         </in>
         <xslt:text>
</xslt:text>
         <in>
            <xslt:value-of select="$values[6]"/>
            <xslt:text>|</xslt:text>
            <xslt:value-of select="$values[6] instance of xs:integer"/>
         </in>
         <xslt:text>
</xslt:text>
         <in>
            <xslt:value-of select="$values[7]"/>
            <xslt:text>|</xslt:text>
            <xslt:value-of select="$values[7] instance of xs:decimal"/>
         </in>
         <xslt:text>
</xslt:text>
         <in>
            <xslt:value-of select="$values[8]"/>
            <xslt:text>|</xslt:text>
            <xslt:value-of select="$values[8] instance of xs:integer"/>
         </in>
         <xslt:text>
</xslt:text>
         <in>
            <xslt:value-of select="$values[9]"/>
            <xslt:text>|</xslt:text>
            <xslt:value-of select="$values[9] instance of xs:integer"/>
         </in>
         <xslt:text>
</xslt:text>
      </out>
   </xslt:template>
</xslt:transform>
