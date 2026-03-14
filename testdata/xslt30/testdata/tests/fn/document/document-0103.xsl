<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test document($uri-sequence, $base-node) function. Second argument is a node in 
  				the input file. First argument is a sequence of xs:string:
				-a sequence of strings
				-a sequence of variables
				-mixed sequence of strings, nodes and variables
				-a subsequence  -->

   <t:variable name="ea" select="xs:string('doc02.xml')"/>

   <t:variable name="doc" select="xs:string('doc01.xsl')"/>
   
   <t:strip-space elements="*"/>

   <t:template name="a">
      <a>
         <t:for-each select="document(('doc02.xml','doc03.xml','doc01.xsl','doc04.xml'), //str1)">
            <t:sort select="document-uri(.)"/>
            <t:copy-of select=".//(body[1]|out)"/>
         </t:for-each>
      </a>
   </t:template>

   <t:template name="b">
      <b>
         <t:for-each select="document(($ea, $doc), //str1)">
            <t:sort select="document-uri(.)"/>
            <t:copy-of select=".//(body[1]|out)"/>
         </t:for-each>
      </b>
   </t:template>

   <t:template name="c">
      <c>
         <t:for-each
            select="document((xs:string('doc02.xml'),xs:string(//str2/@href),xs:string(//str3/@href),'doc04.xml',$ea), //str1)">
            <t:sort select="document-uri(.)"/>
            <t:copy-of select=".//(body[1]|out)"/>
         </t:for-each>
      </c>
   </t:template>

   <t:template name="d">
      <d>
         <t:for-each
            select="document(subsequence((xs:string('doc02.xml'),xs:string(//str2/@href),//str3/@href,'doc04.xml',$ea), 2,1), //str1)">
            <t:sort select="document-uri(.)"/>
            <t:copy-of select=".//(body[1]|out)"/>
         </t:for-each>
      </d>
   </t:template>


</t:transform>
