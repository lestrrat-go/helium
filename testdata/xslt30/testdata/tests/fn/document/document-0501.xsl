<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs t" version="2.0">
   <!-- Purpose: Test document($uri-sequence, $base-node) function. Second argument is a node in 
  				the input file. First argument is a sequence of xs:untypedAtomic:
				-a sequence of nodes
				-a sequence of variables
				-mixed sequence of strings, nodes and variables
				-a subsequence -->

   <t:variable name="ea" select="/doc/str1"/>

   <t:variable name="doc" select="//str3/@href"/>
   
   <t:strip-space elements="*"/>


   <t:template name="a">
      <a>
         <t:copy-of select="doc/document((str1,str2,str3,str4), //str1)//(body[1]|out)"/>
      </a>
   </t:template>
   
   <t:template name="b">
      <b>
         <t:copy-of select="doc/document(($ea, $doc), //str1)//(body[1]|out)"/>
      </b>
   </t:template>
   
   <t:template name="c">
      <c>
         <t:copy-of
            select="doc/document((xs:untypedAtomic('doc02.xml'), str2, str3/@href,'doc04.xml',$ea), //str1)//(body[1]|out)"
         />
      </c>
   </t:template>
   
   <t:template name="d">
      <d>
         <t:copy-of
            select="doc/document(subsequence((str1, str2, str3/@href,'doc04.xml',$ea), 2,1), //str1)//(body[1]|out)"
         />
      </d>
   </t:template>
</t:transform>
