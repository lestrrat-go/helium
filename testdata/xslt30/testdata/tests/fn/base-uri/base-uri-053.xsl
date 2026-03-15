<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="3.0">
<!-- Purpose: test base URI of nodes constructed using built-in template rules -->

   <xsl:variable name="doc1" select="doc('base-uri-001.out')"/>
   <xsl:variable name="doc2" select="doc('base-uri-002.out')"/>
   
   <xsl:template name="xsl:initial-template">
      <out>
         <initial-doc a="{base-uri($doc1)}" b="{base-uri($doc2)}"/>
         <initial-elem a="{base-uri($doc1/*)}" b="{base-uri($doc2/*)}"/>
         
         <xsl:variable name="shallow-copy-docs" as="item()*">
            <xsl:copy select="$doc1"><z/></xsl:copy>
            <xsl:copy select="$doc2"><z/></xsl:copy>
         </xsl:variable>
         <xsl:variable name="shallow-copy-elements" as="item()*">
            <xsl:copy select="$doc1/*"><z/></xsl:copy>
            <xsl:copy select="$doc2/*"><z/></xsl:copy>
         </xsl:variable>
         <shallow-doc a="{base-uri($shallow-copy-docs[1])}" b="{base-uri($shallow-copy-docs[2])}"/>
         <shallow-elem a="{base-uri($shallow-copy-elements[1])}" b="{base-uri($shallow-copy-elements[2])}"/>
         <shallow-doc-deeper a="{base-uri($shallow-copy-docs[1]/z)}" b="{base-uri($shallow-copy-docs[2]/z)}"/>
         <shallow-elem-deeper a="{base-uri($shallow-copy-elements[1]/z)}" b="{base-uri($shallow-copy-elements[2]/z)}"/>
         
         <xsl:variable name="deep-copy-docs" as="item()*">
            <xsl:copy-of select="$doc1"/>
            <xsl:copy-of select="$doc2"/>
         </xsl:variable>
         <xsl:variable name="deep-copy-elements" as="item()*">
            <xsl:copy select="$doc1/*"/>
            <xsl:copy select="$doc2/*"/>
         </xsl:variable>
         <deep-doc a="{base-uri($deep-copy-docs[1])}" b="{base-uri($deep-copy-docs[2])}"/>
         <deep-elem a="{base-uri($deep-copy-elements[1])}" b="{base-uri($deep-copy-elements[2])}"/>
         
         <xsl:variable name="shallow-copy-docs2" as="item()*">
            <xsl:apply-templates select="$doc1" mode="shallow-doc"/>
            <xsl:apply-templates select="$doc2" mode="shallow-doc"/>
         </xsl:variable>
         <xsl:variable name="shallow-copy-elements2" as="item()*">
            <xsl:apply-templates select="$doc1/*" mode="shallow-elem"/>
            <xsl:apply-templates select="$doc2/*" mode="shallow-elem"/>
         </xsl:variable>
         <shallow-copy-doc2 a="{base-uri($shallow-copy-docs2[1])}" b="{base-uri($shallow-copy-docs2[2])}"/>
         <shallow-copy-elem2 a="{base-uri($shallow-copy-elements2[1])}" b="{base-uri($shallow-copy-elements2[2])}"/>
         
         <xsl:variable name="deep-copy-docs2" as="item()*">
            <xsl:apply-templates select="$doc1" mode="deep"/>
            <xsl:apply-templates select="$doc2" mode="deep"/>
         </xsl:variable>
         <xsl:variable name="deep-copy-elements2" as="item()*">
            <xsl:apply-templates select="$doc1/*" mode="deep"/>
            <xsl:apply-templates select="$doc2/*" mode="deep"/>
         </xsl:variable>
         <deep-copy-doc2 a="{base-uri($deep-copy-docs2[1])}" b="{base-uri($deep-copy-docs2[2])}"/>
         <deep-copy-elem2 a="{base-uri($deep-copy-elements2[1])}" b="{base-uri($deep-copy-elements2[2])}"/>

      </out>
   </xsl:template>
   
   <xsl:mode name="shallow-doc" on-no-match="shallow-copy"/>
   <xsl:template match="/*" mode="shallow-doc"/>
   
   <xsl:mode name="shallow-elem" on-no-match="shallow-copy"/>
   <xsl:template match="*/*" mode="shallow-elem"/>
     
   <xsl:mode name="deep" on-no-match="deep-copy"/>

</xsl:transform>
