<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test document($uri-sequence, $base-node) function. Second argument is a node in 
  				the input file. First argument is a sequence of xs:anyURI:
				-a sequence of explicitly constructed literals
				-a sequence of typed variables
				-mixed sequence of strings, nodes and variables
				-a subsequence -->

   <t:variable name="ea" select="/doc/str1/@href" as="xs:anyURI"/>

   <t:variable name="doc" select="/doc/str3/@href" as="xs:anyURI"/>

   <t:strip-space elements="*"/>

   <t:template name="a">
      <a>
         <t:copy-of
            select="document((xs:anyURI('doc02.xml'),xs:anyURI('doc03.xml'),xs:anyURI('doc01.xsl'),xs:anyURI('doc04.xml')), //str1)//(body[1]|out)"/>
      </a>
   </t:template>

   <t:template name="b">
      <b>
         <t:copy-of select="document(($ea, $doc), //str1)//(body[1]|out)"/>
      </b>
   </t:template>

   <t:template name="c">
      <c>
         <t:copy-of
            select="document((xs:anyURI('doc02.xml'),xs:anyURI(//str2/@href),xs:anyURI(//str3/@href),xs:anyURI('doc04.xml'),$ea), //str1)//(body[1]|out)"/>
      </c>
   </t:template>

   <t:template name="d">
      <d>
         <t:copy-of
            select="document(subsequence((xs:anyURI('doc02.xml'),xs:anyURI(//str2/@href),//str3/@href,xs:anyURI('doc04.xml'),$ea), 2,1), //str1)//(body[1]|out)"/>
      </d>
   </t:template>

</t:transform>
