<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of fn:data accessor with sequence of atomic values -->

   <xslt:template match="/">
      <out>
         <xslt:variable name="seq1"
                        select="data( (xs:string('test string'), xs:boolean('1'), xs:float('1E4'), xs:dateTime('1999-08-22T10:05:06')))"/>
         <xslt:variable name="temp1" select="xs:string('test_string')"/>
         <xslt:variable name="temp2" select="xs:boolean('1')"/>
         <xslt:variable name="temp3" select="xs:float('1E4')"/>
         <xslt:variable name="temp4" select="xs:dateTime('1999-08-22T10:05:06')"/>
         <xslt:variable name="seq2" select="data( ($temp1, $temp2, $temp3, $temp4) )"/>
         <t1>
            <xslt:value-of select="data(xs:string('test_string'))"/>
         </t1>
         <t1>
            <xslt:value-of select="data(xs:boolean('1'))"/>
         </t1>
         <t1>
            <xslt:value-of select="data(xs:float('1E4'))"/>
         </t1>
         <t1>
            <xslt:value-of select="data(xs:dateTime('1999-08-22T10:05:06'))"/>
         </t1>
         <t2>
            <xslt:value-of select="$seq1"/>
         </t2>
         <t2>
            <xslt:value-of select="subsequence($seq1, 1, 1)"/>
         </t2>
         <t2>
            <xslt:value-of select="subsequence($seq1, 2, 1)"/>
         </t2>
         <t2>
            <xslt:value-of select="subsequence($seq1, 3, 1)"/>
         </t2>
         <t2>
            <xslt:value-of select="subsequence($seq1, 4, 1)"/>
         </t2>
         <t3>
            <xslt:value-of select="$seq2"/>
         </t3>
         <t3>
            <xslt:value-of select="subsequence($seq2, 1, 1)"/>
         </t3>
         <t3>
            <xslt:value-of select="subsequence($seq2, 2, 1)"/>
         </t3>
         <t3>
            <xslt:value-of select="subsequence($seq2, 3, 1)"/>
         </t3>
         <t3>
            <xslt:value-of select="subsequence($seq2, 4, 1)"/>
         </t3>
      </out>
  
   </xslt:template>
</xslt:transform>
