<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test contructor functions for derived built-in types. The argument is a string/numeric literal -->

   <xslt:template match="/">
      <out>
         <xslt:text>
</xslt:text>
         <e1>
            <xslt:value-of select="xs:normalizedString('like,')"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="xs:normalizedString('like,') instance of xs:normalizedString"/>
         </e1>
         <xslt:text>
</xslt:text>
         <e2>
            <xslt:value-of select="xs:token('two organizations')"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="xs:token('two organizations') instance of xs:token"/>
         </e2>
         <xslt:text>
</xslt:text>
         <e3>
            <xslt:value-of select="xs:language('EN')"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="xs:language('EN') instance of xs:language"/>
         </e3>
         <xslt:text>
</xslt:text>
         <e4>
            <xslt:value-of select="xs:Name('a:_underscore_')"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="xs:Name('a:_underscore_') instance of xs:Name"/>
         </e4>
         <xslt:text>
</xslt:text>
         <e5>
            <xslt:value-of select="xs:NCName('title12')"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="xs:NCName('title12') instance of xs:NCName"/>
         </e5>
         <xslt:text>
</xslt:text>
         <e6>
            <xslt:value-of select="xs:NMTOKEN('business-enforcement::link-chain.common')"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="xs:NMTOKEN('business-enforcement::link-chain.common') instance of xs:NMTOKEN"/>
         </e6>
         <xslt:text>
</xslt:text>
         <e8>
            <xslt:value-of select="xs:ID('A1')"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="xs:ID('A1') instance of xs:ID"/>
         </e8>
         <xslt:text>
</xslt:text>
         <e9>
            <xslt:value-of select="xs:IDREF('a2')"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="xs:IDREF('a2') instance of xs:IDREF"/>
         </e9>
         <xslt:text>
</xslt:text>
         <e11>
            <xslt:value-of select="xs:nonPositiveInteger(-76324602)"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="xs:nonPositiveInteger(-76324602) instance of xs:nonPositiveInteger"/>
         </e11>
         <xslt:text>
</xslt:text>
         <e12>
            <xslt:value-of select="xs:positiveInteger(184467)"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="xs:positiveInteger(184467) instance of xs:positiveInteger"/>
         </e12>
         <xslt:text>
</xslt:text>
         <e12>
            <xslt:value-of select="xs:nonNegativeInteger(444444444)"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="xs:nonNegativeInteger(444444444) instance of xs:nonNegativeInteger"/>
         </e12>
         <xslt:text>
</xslt:text>
         <e13>
            <xslt:value-of select="xs:negativeInteger(-74146)"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="xs:negativeInteger(-74146) instance of xs:negativeInteger"/>
         </e13>
         <xslt:text>
</xslt:text>
         <e14>
            <xslt:value-of select="xs:long(-922337203)"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="xs:long(-922337203) instance of xs:long"/>
         </e14>
         <xslt:text>
</xslt:text>
         <e15>
            <xslt:value-of select="xs:int(131072)"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="xs:int(131072) instance of xs:int"/>
         </e15>
         <xslt:text>
</xslt:text>
         <e16>
            <xslt:value-of select="xs:unsignedLong(184467)"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="xs:unsignedLong(184467) instance of xs:unsignedLong"/>
         </e16>
         <xslt:text>
</xslt:text>
         <e17>
            <xslt:value-of select="xs:unsignedInt(573231403)"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="xs:unsignedInt(573231403) instance of xs:unsignedInt"/>
         </e17>
         <xslt:text>
</xslt:text>
         <e18>
            <xslt:value-of select="xs:short(-32768)"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="xs:short(-32768) instance of xs:short"/>
         </e18>
         <xslt:text>
</xslt:text>
         <e19>
            <xslt:value-of select="xs:unsignedShort(65535)"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="xs:unsignedShort(65535) instance of xs:unsignedShort"/>
         </e19>
         <xslt:text>
</xslt:text>
         <e20>
            <xslt:value-of select="xs:byte(-128)"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="xs:byte(-128) instance of xs:byte"/>
         </e20>
         <xslt:text>
</xslt:text>
         <e21>
            <xslt:value-of select="xs:unsignedByte(110)"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="xs:unsignedByte(110) instance of xs:unsignedByte"/>
         </e21>
         <xslt:text>
</xslt:text>
         <e22>
            <xslt:value-of select="xs:ENTITY('ent')"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="xs:ENTITY('ent') instance of xs:ENTITY"/>
         </e22>
      </out>
   </xslt:template>
</xslt:transform>
