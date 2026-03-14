<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" version="3.0">

    <?spec xslt#patterns?>
    <!-- Purpose: numeric value of predicate.  -->
    <!-- Purpose: comments in a pattern.  -->
    <!-- see Bug 28336 -->

    <xsl:template name="xsl:initial-template">
        <out>
            <xsl:apply-templates select="17" />
        </out>
    </xsl:template>


    <xsl:template match=".[. instance of xs:integer]">
        <!-- has same priority, even with multiple predicates, will match secondly -->
        <second>
            <xsl:next-match />
        </second>
    </xsl:template>

    <xsl:template match="(: also with :) . (:silly (: comments :) :) [ (:for the :). instance (:second :) of (:match:) xs:integer][ (:predicate of '1' always matches:) 1] (:really?:)">
        <first>
            <xsl:next-match />
        </first>
    </xsl:template>

    <xsl:template match="(:a(:z:)a:) . (:b:) [. instance (:c:) of xs:integer](:d:)[2](:e:)">
        <!-- quote from spec: "A predicate with the numeric value 1 (one) always matches, and a predicate with any other numeric value never matches." -->
        <should-not-match>
            <xsl:next-match />
        </should-not-match>
    </xsl:template>

</xsl:stylesheet>
