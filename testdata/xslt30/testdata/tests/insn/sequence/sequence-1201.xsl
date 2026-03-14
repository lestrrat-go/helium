<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:example1="http://www.example14.com"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs example1">
<!-- Purpose: sequence constructor with explicit items of all types supported by basic processor -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:template match="doc">
      <xslt:text>
</xslt:text>
      <out>
         <xslt:text>
</xslt:text>
         <xslt:value-of select="(xs:integer('1'), xs:decimal('2.2'), xs:double('3.3E0'), xs:string('four'), xs:date('5555-05-05'), xs:time('06:06:06'), xs:dateTime('7777-07-07T07:07:07+07:00'),xs:yearMonthDuration('P8Y8M'), xs:anyURI('http://www.999.org/999/rfc999.txt'),xs:untypedAtomic('ten'),xs:dayTimeDuration('P11DT11H11M11S'),xs:boolean('false'),num)"
                        separator=", "/>
         <xslt:text>
</xslt:text>
         <xslt:value-of select="namespace-uri-from-QName(subsequence((xs:integer('1'), xs:decimal('2.2'), xs:double('3.3E0'), xs:string('four'), xs:date('5555-05-05'), xs:time('06:06:06'), xs:dateTime('7777-07-07T07:07:07+07:00'),xs:yearMonthDuration('P8Y8M'), xs:anyURI('http://www.999.org/999/rfc999.txt'),xs:untypedAtomic('ten'),xs:dayTimeDuration('P11DT11H11M11S'),xs:boolean('false'),xs:QName('example1:foo14'),num),num,1))"
                        separator=", "/>
         <xslt:text>:</xslt:text>
         <xslt:value-of select="local-name-from-QName(subsequence((xs:integer('1'), xs:decimal('2.2'), xs:double('3.3E0'), xs:string('four'), xs:date('5555-05-05'), xs:time('06:06:06'), xs:dateTime('7777-07-07T07:07:07+07:00'),xs:yearMonthDuration('P8Y8M'), xs:anyURI('http://www.999.org/999/rfc999.txt'),xs:untypedAtomic('ten'),xs:dayTimeDuration('P11DT11H11M11S'),xs:boolean('false'),xs:QName('example1:foo14'),num),num,1))"
                        separator=", "/>
         <xslt:text>
</xslt:text>
      </out>
      <xslt:text>
</xslt:text>
   </xslt:template>
</xslt:transform>
