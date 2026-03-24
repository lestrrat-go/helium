<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                xml:base="http://www.baseuri.exmpl/tests/"
                version="2.0">
<!-- Purpose: Testcase with fn:resolve-uri($relative) where $relative is a relative URI. Different argument syntax constructs are tested.-->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:template match="/doc" xml:base="http://a/b/c/d;p?q">
      <output xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <xslt:text>
</xslt:text>
         <out1>
            <xslt:value-of select="resolve-uri('g:h')"/>
         </out1>
         <xslt:text>
</xslt:text>
         <out2>
            <xslt:value-of select="resolve-uri('g')"/>
         </out2>
         <xslt:text>
</xslt:text>
         <out3>
            <xslt:value-of select="resolve-uri('./g')"/>
         </out3>
         <xslt:text>
</xslt:text>
         <out4>
            <xslt:value-of select="resolve-uri('g/')"/>
         </out4>
         <xslt:text>
</xslt:text>
         <out5>
            <xslt:value-of select="resolve-uri('/g')"/>
         </out5>
         <xslt:text>
</xslt:text>
         <out6>
            <xslt:value-of select="resolve-uri('//g')"/>
         </out6>
         <xslt:text>
</xslt:text>
         <out8>
            <xslt:value-of select="resolve-uri('g?y')"/>
         </out8>
         <xslt:text>
</xslt:text>
         <out9>
            <xslt:value-of select="resolve-uri('#g')"/>
         </out9>
         <xslt:text>
</xslt:text>
         <out10>
            <xslt:value-of select="resolve-uri('g#s')"/>
         </out10>
         <xslt:text>
</xslt:text>
         <out11>
            <xslt:value-of select="resolve-uri('g?y#s')"/>
         </out11>
         <xslt:text>
</xslt:text>
         <out12>
            <xslt:value-of select="resolve-uri(';x')"/>
         </out12>
         <xslt:text>
</xslt:text>
         <out13>
            <xslt:value-of select="resolve-uri('g;x')"/>
         </out13>
         <xslt:text>
</xslt:text>
         <out14>
            <xslt:value-of select="resolve-uri('g;x?y#s')"/>
         </out14>
         <xslt:text>
</xslt:text>
         <out15>
            <xslt:value-of select="resolve-uri('.')"/>
         </out15>
         <xslt:text>
</xslt:text>
         <out16>
            <xslt:value-of select="resolve-uri('./')"/>
         </out16>
         <xslt:text>
</xslt:text>
         <out17>
            <xslt:value-of select="resolve-uri('..')"/>
         </out17>
         <xslt:text>
</xslt:text>
         <out18>
            <xslt:value-of select="resolve-uri('../')"/>
         </out18>
         <xslt:text>
</xslt:text>
         <out19>
            <xslt:value-of select="resolve-uri('../g')"/>
         </out19>
         <xslt:text>
</xslt:text>
         <out20>
            <xslt:value-of select="resolve-uri('../..')"/>
         </out20>
         <xslt:text>
</xslt:text>
         <out21>
            <xslt:value-of select="resolve-uri('../../')"/>
         </out21>
         <xslt:text>
</xslt:text>
         <out22>
            <xslt:value-of select="resolve-uri('../../g')"/>
         </out22>
      </output>
   </xslt:template>
</xslt:transform>
