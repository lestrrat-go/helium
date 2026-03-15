<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:f="http://local-functions.org/"
    exclude-result-prefixes="map xs f">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:param name="STREAMABLE" select="true()" static="yes"/>
    
    <xsl:import-schema schema-location="books.xsd"/>
    
    <!-- Validation = strict -->
    
    <xsl:template name="s-101" use-when="$RUN">
      <out>
        <xsl:source-document _streamable="{$STREAMABLE}" href="books.xml" validation="strict">
          <xsl:value-of select="//PRICE instance of element(*, xs:decimal)+"/>
        </xsl:source-document>
      </out>
    </xsl:template>
    
    <!-- Validation = strict, invalid -->
    
    <xsl:template name="s-102" use-when="$RUN">
      <out>
        <xsl:source-document _streamable="{$STREAMABLE}" href="books-invalid.xml" validation="strict">
          <xsl:value-of select="//PRICE instance of element(*, xs:decimal)+"/>
        </xsl:source-document>
      </out>
    </xsl:template>
    
    <!-- Validation by type, valid -->
    
    <xsl:template name="s-103" use-when="$RUN">
      <out>
        <xsl:source-document _streamable="{$STREAMABLE}" href="books.xml" type="BOOKLIST-type">
          <xsl:value-of select="//PRICE instance of element(*, xs:decimal)+"/>
        </xsl:source-document>
      </out>
    </xsl:template>
    
    <!-- Validation by type, invalid -->
    
    <xsl:template name="s-104" use-when="$RUN">
      <out>
        <xsl:source-document _streamable="{$STREAMABLE}" href="books-invalid.xml" type="BOOKLIST-type">
          <xsl:value-of select="//PRICE instance of element(*, xs:decimal)+"/>
        </xsl:source-document>
      </out>
    </xsl:template>
    
    <!-- Partial validation, valid -->
    
    <xsl:template name="s-105" use-when="$RUN">
      <out>
        <xsl:source-document _streamable="{$STREAMABLE}" href="books-invalid.xml">
          <xsl:copy-of select="/BOOKLIST/BOOKS/ITEM[1]" validation="strict"/>
        </xsl:source-document>
      </out>
    </xsl:template>
    
    <!-- Partial validation, invalid -->
    
    <xsl:template name="s-106" use-when="$RUN">
      <out>
        <xsl:source-document _streamable="{$STREAMABLE}" href="books-invalid.xml">
          <xsl:copy-of select="/BOOKLIST/BOOKS/ITEM[2]" validation="strict"/>
        </xsl:source-document>
      </out>
    </xsl:template>
    
    <!-- xsl:source-document called within a function -->
    
    <xsl:template name="s-107" use-when="$RUN">
      <out>
        <xsl:value-of select="round(avg(f:read-stream('books.xml')), 2)"/>
      </out>
    </xsl:template>
    
    <xsl:function name="f:read-stream" as="xs:decimal*">
      <xsl:param name="uri" as="xs:string"/>
      <xsl:source-document _streamable="{$STREAMABLE}" href="{$uri}">
        <xsl:sequence select="//PRICE/data(.)"/>
      </xsl:source-document>
    </xsl:function>
    
    <!-- xsl:source-document called within a function with early termination -->
    
    <xsl:template name="s-108" use-when="$RUN">
      <out>
        <xsl:value-of select="exists(f:read-stream('books.xml'))"/>
      </out>
    </xsl:template>
    
    <!-- xsl:source-document called within a function using a constructor function -->
    
    <xsl:template name="s-109" use-when="$RUN">
      <out>
        <xsl:value-of select="round(avg(f:read-stream-decimal('books.xml')), 2)"/>
      </out>
    </xsl:template>
    
    <xsl:function name="f:read-stream-decimal" as="xs:decimal*">
      <xsl:param name="uri" as="xs:string"/>
      <xsl:source-document _streamable="{$STREAMABLE}" href="{$uri}">
        <xsl:sequence select="//PRICE/xs:decimal(.)"/>
      </xsl:source-document>
    </xsl:function>    
    


              
    
</xsl:stylesheet>